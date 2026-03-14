<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <!-- xsl:merge test using accumulators. -->
    
    <xsl:param name="STREAMABLE" static="yes" select="false()"/>
    <xsl:output method="xml" indent="no"/>
    <xsl:strip-space elements="*"/>
    <xsl:template match="/">
        
        <events>
            <xsl:merge>
                <xsl:merge-source for-each-source="'log-file-1.xml'" select="events/event" _streamable="{$STREAMABLE}" use-accumulators="path">
                        <xsl:merge-key select="xs:dateTime(@timestamp)"/>
                </xsl:merge-source>
                <xsl:merge-source for-each-source="'log-file-2.xml'" select="log/day/record" _streamable="{$STREAMABLE}" use-accumulators="path">
                        <xsl:merge-key select="dateTime(../@date, time)"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <group>
                        <xsl:for-each select="current-merge-group()">
                          <item><xsl:value-of select="accumulator-before('path')"/></item>
                        </xsl:for-each>  
                        <xsl:copy-of select="current-merge-group()" />
                    </group>
                </xsl:merge-action>
            </xsl:merge>
        </events>
        
    </xsl:template>
    
    <xsl:accumulator name="path" initial-value="''" _streamable="{$STREAMABLE}">
      <xsl:accumulator-rule match="*" select="concat(base-uri(/), ' - ', string-join(ancestor::*/name(), '/'))"/>
    </xsl:accumulator>  
</xsl:stylesheet>
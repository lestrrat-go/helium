<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <!-- xsl:merge test using accumulators. -->
    
    <xsl:output method="xml" indent="no"/>
    <xsl:strip-space elements="*"/>
    <xsl:param name="STREAMABLE" select="false()" static="true"/>
    <xsl:template match="/">
        
        <events>
            <xsl:merge>
                <xsl:merge-source name="s-1" for-each-source="'log-file-1.xml'" select="events/event" 
                    use-accumulators="interval-1" _streamable="{$STREAMABLE}">
                    <xsl:merge-key select="xs:dateTime(@timestamp)"/>
                </xsl:merge-source>
                <xsl:merge-source name="s-2" for-each-source="'log-file-2.xml'" select="log/day/record" 
                    use-accumulators="interval-2" _streamable="{$STREAMABLE}">
                    <xsl:merge-key select="dateTime(../@date, time)"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <group>
                        <xsl:for-each select="current-merge-group('s-1')">
                            <interval in="log-1"><xsl:value-of select="accumulator-after('interval-1')"/></interval>
                        </xsl:for-each>
                        <xsl:for-each select="current-merge-group('s-2')">
                            <interval in="log-2"><xsl:value-of select="accumulator-after('interval-2')"/></interval>
                        </xsl:for-each>
                        <xsl:copy-of select="current-merge-group()" />
                    </group>
                </xsl:merge-action>
            </xsl:merge>
        </events>
        
    </xsl:template>
    
    <xsl:variable name="ORIGIN" select="xs:dateTime('2009-01-01T00:00:00Z')"/>
    
    <xsl:accumulator name="latest-1" initial-value="map{0:$ORIGIN, 1:$ORIGIN}" as="map(*)"
        _streamable="{$STREAMABLE}">
        <xsl:accumulator-rule match="event" select="map{0:$value?1, 1:xs:dateTime(@timestamp)}"/>
    </xsl:accumulator>
    
    <xsl:accumulator name="latest-2" initial-value="map{0:$ORIGIN, 1:$ORIGIN}" as="map(*)" _streamable="{$STREAMABLE}">
        <xsl:accumulator-rule match="time" select="map{0:$value?1, 1:dateTime(xs:date(../../@date), xs:time(.))}"/>
    </xsl:accumulator>
    
    <xsl:accumulator name="interval-1" initial-value="0" _streamable="{$STREAMABLE}">
      <xsl:accumulator-rule match="event" select="let $m := accumulator-before('latest-1') return ($m?1 - $m?0)"/>
    </xsl:accumulator>
    
    <xsl:accumulator name="interval-2" initial-value="0" _streamable="{$STREAMABLE}">
        <xsl:accumulator-rule match="time" select="let $m := accumulator-before('latest-2') return ($m?1 - $m?0)"/>
    </xsl:accumulator>
</xsl:stylesheet>
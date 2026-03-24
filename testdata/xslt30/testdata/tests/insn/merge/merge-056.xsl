<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:output method="xml" indent="no"/>
    <xsl:strip-space elements="*"/>
    <xsl:variable name="doc1" select="doc('log-file-1.xml')" />
    <xsl:variable name="asc_or_desc" select="$doc1/events/myOrder/@order" />
    <xsl:variable name="data-type" select="$doc1/events/myDataType/@data-type" />
    <xsl:template match="/">
        
        <events>
            <xsl:merge xml:base="../../">
                <xsl:merge-source select="$doc1/events/event">
                        <xsl:merge-key select="xs:dateTime(@timestamp)" order="{$asc_or_desc}" data-type="{$data-type}"/>
                </xsl:merge-source>
                <xsl:merge-source for-each-source="'insn/merge/log-file-2.xml'" select="log/day/record" validation="strip" streamable="yes">
                        <xsl:merge-key select="dateTime(../@date, time)" order='ascending' data-type="text"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <group>
                        <xsl:call-template name="n"/>
                    </group>
                </xsl:merge-action>
            </xsl:merge>
        </events>
    
    </xsl:template>
    
    <xsl:template name="n">
      <xsl:value-of select="current-merge-key()"/>
    </xsl:template>
</xsl:stylesheet>
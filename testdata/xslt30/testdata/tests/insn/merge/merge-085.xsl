<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <!-- xsl:merge test on a homogeneous collection of XML log files. -->
    <xsl:output method="xml" indent="no"/>
    
    <xsl:template name="main">
        <events>
            <xsl:merge>                
                <xsl:merge-source 
                		for-each-item="collection('log-files')"
                		select="events/event">
                    <xsl:merge-key select="@timestamp"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:copy-of select="current-merge-group()" />
                </xsl:merge-action>
                <xsl:fallback>111</xsl:fallback>
                <xsl:fallback>222</xsl:fallback>
            </xsl:merge>
            
        </events>
    </xsl:template>
</xsl:stylesheet>
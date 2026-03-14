<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:template match="/">
        <!-- xsl:merge-source with more than one xsl:merge-input child element - error -->
        
    <xsl:result-document href="merged-events.xml">
        <events>
            <xsl:merge>
                <xsl:merge-source select="doc('log-file-1.xml')">
                    
                    <xsl:merge-input select="events/event">
                        <xsl:merge-key select="@timestamp"/>
                    </xsl:merge-input>                
                
                    <xsl:merge-input select="log/day/record">
                        <xsl:merge-key select="dateTime(../@date, time)"/>
                    </xsl:merge-input>
                    
                </xsl:merge-source>
                
                <xsl:merge-action>
                    <xsl:apply-templates select="current-merge-group()" 
                        mode="standardize-log-entry" />
                </xsl:merge-action>
                
            </xsl:merge>
        </events>
    </xsl:result-document>
    </xsl:template>
</xsl:stylesheet>
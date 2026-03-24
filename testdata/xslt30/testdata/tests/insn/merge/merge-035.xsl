<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:template match="/">
        <!-- xsl:merge with a disallowed child element -->
        
    <xsl:result-document href="merged-events.xml">
        <events>
            <xsl:merge>
                
                <xsl:merge-source select="collection('log-file-1')">
                    <xsl:merge-input select="events/event">
                        <xsl:merge-key select="@timestamp"/>
                        
                    </xsl:merge-input>
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:copy-of select="current-merge-group()" />
                </xsl:merge-action>
                <xsl:comment>disallowed</xsl:comment>
            </xsl:merge>
            
        </events>
    </xsl:result-document>
        </xsl:template>
</xsl:stylesheet>
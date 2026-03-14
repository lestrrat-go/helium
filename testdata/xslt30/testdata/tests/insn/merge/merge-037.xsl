<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:template name="main">
    <xsl:result-document href="merged-events.xml">
        <events>
            <xsl:merge>
                <xsl:merge-source>

                        
                        <xsl:merge-key select="dateTime(current-date(), @time)" />
                        <xsl:perform-sort select="doc('error-log.xml')//error">
                            <xsl:sort select="@time"/>
                        </xsl:perform-sort>
                         

                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:apply-templates select="current-merge-group()" />
                </xsl:merge-action>
            </xsl:merge>
        </events>
    </xsl:result-document>
    </xsl:template>
</xsl:stylesheet>
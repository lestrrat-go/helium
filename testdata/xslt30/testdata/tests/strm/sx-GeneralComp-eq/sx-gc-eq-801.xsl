<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"                 
    version="3.0">
    
    <xsl:output method="xml" indent="yes"/>
    
    <xsl:mode streamable="true"/>
    
    <xsl:template match="ProteinDatabase">
        <ProteinEntry>
            <xsl:iterate select="ProteinEntry">
                <xsl:choose>
                    <xsl:when test="header/uid = 'CCHU'">                  
                        <xsl:value-of select="'hello'"/>                  
                    </xsl:when>
                    <xsl:otherwise>
                        <xsl:break/>
                    </xsl:otherwise>
                </xsl:choose>
            </xsl:iterate>
        </ProteinEntry>
    </xsl:template>
    
</xsl:stylesheet>
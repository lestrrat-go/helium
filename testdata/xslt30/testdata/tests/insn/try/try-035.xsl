<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"                
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="err"
    version="3.0">
    
    <xsl:output method="xml" indent="yes"/>                
    
    <xsl:template match="/">        
        <xsl:try>
            <xsl:value-of select="for $x in (1 to count(//*)) return (1 div 0)"/>
            <xsl:catch errors="*">
                <error code="{$err:code}" description="{$err:description}" location="line:{$err:line-number}, col:{$err:column-number}"/>
            </xsl:catch>
        </xsl:try>       
    </xsl:template>
    
</xsl:stylesheet>
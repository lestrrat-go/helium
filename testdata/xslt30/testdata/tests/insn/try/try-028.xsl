<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="xs">

    <xsl:param name="p" select="0"/>
    <xsl:variable name="q" select="22 div $p"/>
    
    <xsl:template match="/" name="main">
        <output>
            
                <xsl:try>
                    <out><xsl:value-of select="$q"/></out>
                    <xsl:catch>
                        <caught/>
                    </xsl:catch>
                </xsl:try>

        </output>
    </xsl:template>
</xsl:stylesheet>

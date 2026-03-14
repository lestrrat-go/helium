<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="xs">

    <xsl:param name="p" select="10"/>
    <xsl:param name="root" select="/"/>
    
    <xsl:template match="/" name="main">
        <output>
           <xsl:for-each select="1 to $p">
                <xsl:variable name="q" select="$root//*[position() = 22 div (current() - 1)]"/>
                <xsl:try>
                    <out><xsl:value-of select="$q"/></out>
                    <xsl:catch>
                        <caught/>
                    </xsl:catch>
                </xsl:try>
           </xsl:for-each>
        </output>
    </xsl:template>
</xsl:stylesheet>

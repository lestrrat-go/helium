<?xml version="1.0" encoding="ISO-8859-1"?> 
<xsl:stylesheet 
        xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
        <!-- context item within matching-substring is not a node -->

<xsl:template match="/"> 
<ales> 
        <xsl:apply-templates/> 
</ales> 
</xsl:template> 
<xsl:template match="ales/text"> 
        <xsl:analyze-string select="."  regex ="\s*Codi\s*:\s*"> 
                <xsl:matching-substring> 
                        <codi><xsl:value-of select="following-sibling::text[position()=1]"/></codi> 
                </xsl:matching-substring> 
                <xsl:non-matching-substring> 
                          <no-match><xsl:value-of select="."/></no-match> 
                </xsl:non-matching-substring> 
          </xsl:analyze-string>         
</xsl:template> 
</xsl:stylesheet>
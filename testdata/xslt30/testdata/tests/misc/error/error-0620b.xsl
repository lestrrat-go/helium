<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a variable-binding element has a
                        select attribute and has non-empty content.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0620?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates>
            <xsl:with-param name="p" select="2">fred</xsl:with-param>
        </xsl:apply-templates>
      </out>
  </xsl:template>
  
  <xsl:template match="doc">
      <xsl:param name="p" as="xs:integer"/>
      <xsl:value-of select="$p"/>
  </xsl:template>   
  


</xsl:stylesheet>

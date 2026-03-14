<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the conversion of
                     the supplied value of a parameter to
                     its required type fails.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE0590?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates>
            <xsl:with-param name="p" select="current-date()"/>
        </xsl:apply-templates>
      </out>
  </xsl:template>
  
  <xsl:template match="doc">
      <xsl:param name="p" as="xs:integer"/>
      <xsl:value-of select="$p"/>
  </xsl:template>   
  


</xsl:stylesheet>

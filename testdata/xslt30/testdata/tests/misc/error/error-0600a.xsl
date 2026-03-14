<?xml version="1.0" encoding="UTF-8"?>
<!--If a default value is given explicitly, that is, if there is either a
                        select attribute or a non-empty sequence constructor, then it is a
                        type error if the default value
                     cannot be converted to the required type, using the function conversion rules.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE0600?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates>
          
        </xsl:apply-templates>
      </out>
  </xsl:template>
  
  <xsl:template match="doc">
      <xsl:param name="p" as="xs:integer" select="3.5"/>
      <xsl:value-of select="$p"/>
  </xsl:template>   
  


</xsl:stylesheet>

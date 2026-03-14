<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the values of
                     the name attribute of two sibling
                           xsl:param elements represent the same expanded QName.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0580?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
  <xsl:function name="my:f">
      <xsl:param name="p"/>
      <xsl:param name="p"/> 
      <xsl:value-of select="$p"/>
  </xsl:function>   
  


</xsl:stylesheet>

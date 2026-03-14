<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                           xsl:key declaration has a use attribute and
                        has non-empty content, or if it has empty content and no use
                        attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1205?>

   <xsl:key name="k" match="*"/>


  <xsl:template name="main">
      <out>
         <xsl:sequence select="3"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>

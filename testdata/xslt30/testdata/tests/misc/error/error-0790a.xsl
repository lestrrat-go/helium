<?xml version="1.0" encoding="UTF-8"?>
<!--If the value of a parameter to a stylesheet function cannot be converted to the required type, a
                        type error is signaled.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE0790?>



  <xsl:template name="main">
      <out>
        <xsl:sequence select="my:f(2)"/>
      </out>
  </xsl:template>
  
   <xsl:function name="my:f">
      <xsl:param name="x" as="xs:date"/>
      <xsl:sequence select="$x"/>
   </xsl:function>



</xsl:stylesheet>

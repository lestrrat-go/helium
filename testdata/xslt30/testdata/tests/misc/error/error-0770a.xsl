<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error for a stylesheet to contain two or more functions
                     with the same expanded QName, the
                     same arity, and the same import precedence, unless there is
                     another function with the same expanded QName and arity, and a higher import precedence.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0770?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
   <xsl:function name="my:f">
      <xsl:param name="x"/>
      <xsl:sequence select="2"/>
   </xsl:function>

   <xsl:function name="my:f">
      <xsl:param name="x"/>
      <xsl:sequence select="3"/>
   </xsl:function>


</xsl:stylesheet>

<?xml version="1.0" encoding="UTF-8"?>
<!--If the as attribute of xsl:function
                      is specified, then the result evaluated by the sequence constructor (see ) is converted to the required type, using the
                        function conversion
                           rules. It is a type error
                        if this conversion fails.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE0780?>



  <xsl:template name="main">
      <out>
        <xsl:sequence select="my:f(2)"/>
      </out>
  </xsl:template>
  
   <xsl:function name="my:f" as="xs:date">
      <xsl:param name="x"/>
      <xsl:sequence select="2"/>
   </xsl:function>



</xsl:stylesheet>

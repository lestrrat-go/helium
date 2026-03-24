<?xml version="1.0" encoding="UTF-8"?>
<!--Because arguments to a stylesheet function call must all be
                        specified, the xsl:param elements within an
                        xsl:function element must not specify a
                        default value: this means they must be empty, and
                        must not have a select attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0760?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
   <xsl:function name="my:f">
      <xsl:param name="x">3</xsl:param>
      <xsl:sequence select="2"/>
   </xsl:function>



</xsl:stylesheet>

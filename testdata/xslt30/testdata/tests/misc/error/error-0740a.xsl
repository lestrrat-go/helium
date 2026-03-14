<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if
                     a stylesheet function
                     has a name that is in no namespace.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0740?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
   <xsl:function name="f">
      <xsl:sequence select="2"/>
   </xsl:function>



</xsl:stylesheet>

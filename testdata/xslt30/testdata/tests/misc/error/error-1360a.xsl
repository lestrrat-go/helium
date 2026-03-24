<?xml version="1.0" encoding="UTF-8"?>
<!--If the current function is evaluated within an
                  expression that is evaluated when the context item is absent, a non-recoverable dynamic error
                  occurs.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1360?>


  <xsl:template name="main">
      <out>
        <xsl:sequence select="my:f()"/>
      </out>
  </xsl:template>
  
  <xsl:function name="my:f">
      <xsl:sequence select="name(current())"/>
  </xsl:function>
  



</xsl:stylesheet>

<?xml version="1.0" encoding="UTF-8"?>
<!--In the case of xsl:call-template, it is a static error to pass a non-tunnel
                        parameter named x to a template that does not have a 
                        non-tunnel template parameter named
                           x, unless the
                              xsl:call-template instruction is processed with
                              XSLT 1.0
                           behavior.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0680?>



  <xsl:template name="main">
      <out>
        <xsl:call-template name="x">
            <xsl:with-param name="my:p" select="2"/>
        </xsl:call-template>
      </out>
  </xsl:template>
  
  <xsl:template name="x"/>

  


</xsl:stylesheet>

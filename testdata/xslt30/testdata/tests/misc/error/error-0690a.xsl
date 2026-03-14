<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a template
                        that is invoked using xsl:call-template declares a template parameter specifying
                           required="yes" and not specifying tunnel="yes",
                        if no value for this parameter is supplied by the calling
                           xsl:call-template instruction.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0690?>



  <xsl:template name="main">
      <out>
        <xsl:call-template name="x">
          
        </xsl:call-template>
      </out>
  </xsl:template>
  
  <xsl:template name="x">
      <xsl:param name="x" required="yes"/>
      <xsl:value-of select="2"/>
  </xsl:template>

  


</xsl:stylesheet>

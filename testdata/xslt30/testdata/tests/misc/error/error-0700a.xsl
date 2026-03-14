<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if a template that is invoked using
                        xsl:apply-templates, xsl:apply-imports, or
                        xsl:next-match declares a template parameter with
                        required="yes" and no value for this parameter is supplied by
                     the calling instruction. The same error is reported in the case of a tunnel parameter whether invoked using one of these three
                     instructions or by xsl:call-template, as explained in .-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0700?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
  <xsl:template match="doc">
      <xsl:param name="x" required="yes"/>
      <xsl:value-of select="2"/>
  </xsl:template>

  


</xsl:stylesheet>

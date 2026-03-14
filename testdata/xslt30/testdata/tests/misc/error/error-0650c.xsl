<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a stylesheet contains an
                        xsl:call-template instruction whose name
                     attribute does not match the name attribute of any
                        named template 
                           visible in the containing package 
                           (this includes any template defined in this package, as well as templates
                           accepted from used packages whose visibility in this package
                           is not hidden). For more details of the
                  process of binding the called template, see .-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0650?>



  <xsl:template name="main">
      <out>
        <xsl:call-template name="my:template"/>
      </out>
  </xsl:template>
  


  


</xsl:stylesheet>

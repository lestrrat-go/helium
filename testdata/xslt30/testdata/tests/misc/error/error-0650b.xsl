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
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:call-template - unknown template name -->
<?spec xslt#errors?><?error XTSE0650?>

   <xsl:template name="x"/>
  
  <xsl:template match="doc">
      <out>
         <xsl:call-template name="y"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  


</xsl:stylesheet>

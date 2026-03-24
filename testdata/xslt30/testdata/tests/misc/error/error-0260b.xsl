<?xml version="1.0" encoding="UTF-8"?>
<!--Within an XSLT element that is
                        required to be empty, any content other than comments or
                     processing instructions, including any whitespace text node preserved using the
                        xml:space="preserve" attribute, is a static error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:import - non-empty element -->
<?spec xslt#errors?><?error XTSE0260?>
   <xsl:import href="error-ok.xsl">text</xsl:import>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>

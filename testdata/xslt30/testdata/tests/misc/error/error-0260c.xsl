<?xml version="1.0" encoding="UTF-8"?>
<!--Within an XSLT element that is
                        required to be empty, any content other than comments or
                     processing instructions, including any whitespace text node preserved using the
                        xml:space="preserve" attribute, is a static error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0260?>

  <xsl:strip-space elements="*">*</xsl:strip-space>

  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>




</xsl:stylesheet>

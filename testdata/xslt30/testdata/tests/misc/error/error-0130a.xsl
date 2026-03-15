<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                           xsl:stylesheet element has a child element whose name
                        has a null namespace URI.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0130?>


  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>

  <porridge/>

</xsl:stylesheet>

<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a
                        stylesheet module directly or indirectly includes itself.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:include - circular reference -->
<?spec xslt#errors?><?error XTSE0180?>

   <xsl:include href="error-0180b.xsl"/>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>

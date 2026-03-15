<?xml version="1.0" encoding="UTF-8"?>
<!--An xsl:import element must be a top-level element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:import - not top-level -->
<?spec xslt#errors?><?error XTSE0010?>

   <xsl:variable name="test">
      <xsl:import href="error-0700a.xsl"/>
   </xsl:variable>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>

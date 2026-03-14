<?xml version="1.0" encoding="UTF-8"?>
<!--An xsl:import element must be a top-level element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0190?>


  <xsl:template name="main">
      <xsl:import href="error-0700a.xsl"/>
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>



</xsl:stylesheet>

<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a
                        stylesheet module directly or indirectly imports itself.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:import - self-referential -->
<?spec xslt#errors?><?error XTSE0210?>

   <xsl:import href="error-0210b.xsl"/>

  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>

<?xml version="1.0" encoding="UTF-8"?>
<!-- It is a static error if there is a
                        stylesheet module in the
                        stylesheet that specifies
                        input-type-annotations="strip" and another stylesheet module that specifies
                        input-type-annotations="preserve".-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                input-type-annotations="strip">

<?error XTSE0265?>

  <xsl:include href="error-0265a2.xsl"/>

  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>




</xsl:stylesheet>

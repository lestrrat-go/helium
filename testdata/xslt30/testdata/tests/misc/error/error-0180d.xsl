<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a
                        stylesheet module directly or indirectly includes itself.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0180?>

  <xsl:include href="error-0180d.xsl"/>
  <xsl:template name="main">
    
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>



</xsl:stylesheet>

<?xml version="1.0" encoding="UTF-8"?>
<!--A literal result element
                     that is used as the outermost element of a simplified stylesheet module
                        must have an xsl:version attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?error XTSE0165?>

  <xsl:import href="_error-test-set.xml"/>


  <xsl:template name="main">
      <out>
         <xsl:apply-templates/>
      </out>
  </xsl:template>



</xsl:stylesheet>

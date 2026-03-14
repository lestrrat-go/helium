<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: namedtemplate17 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 6 Named Templates -->
    <!-- Purpose: Test import precedence: both 18a and 18b have "show" template, b wins. -->
  <!-- Output should be 'Template from 18b has been called.' -->

<xsl:import href="call-template-18a.xsl"/>
<xsl:import href="call-template-18b.xsl"/>

<xsl:template match="doc">
  <out>
    <xsl:call-template name="show"/>
  </out>
</xsl:template>

</xsl:stylesheet>
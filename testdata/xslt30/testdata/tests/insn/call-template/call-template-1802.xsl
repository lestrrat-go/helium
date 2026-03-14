<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: namedtemplate18 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 6 Named Templates -->
    <!-- Purpose: Test import precedence: main stylesheet wins over imported one. -->
  <!-- Output should be 'Template from MAIN has been called.' -->

<xsl:import href="call-template-18b.xsl"/>

<xsl:template match="doc">
  <out>
    <xsl:call-template name="show"/>
  </out>
</xsl:template>

<xsl:template name="show">
  <xsl:text>Template from MAIN has been called.</xsl:text>
</xsl:template>

</xsl:stylesheet>
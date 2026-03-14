<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position106 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test for last() on 'ancestor-or-self::' axis. -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:if test="generate-id(ancestor-or-self::node()[last()])=generate-id(/)">Success</xsl:if>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
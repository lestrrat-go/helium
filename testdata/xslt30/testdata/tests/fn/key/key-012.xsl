<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: key121 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 12.2 -->
    <!-- Purpose: Test for nested calls to key() function. -->

<xsl:key name="titles" match="div" use="title" />
<xsl:key name="marks" match="div" use="@mark" />

<xsl:template match="doc">
 <root>
  The next key finds three divisions:
  <xsl:for-each select="key('marks',key('titles', 'Expressions')/finder)">
    <xsl:value-of select="p"/>
  </xsl:for-each>
 </root>
</xsl:template>

</xsl:stylesheet>

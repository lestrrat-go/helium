<?xml version="1.1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec serialization#xml-version?>
  <!-- PURPOSE: Generate a document that's valid in XML 1.1 but not in XML 1.0 -->
  <!-- the output of this is the stdxmlfile.xml used as input to other tests -->

<xsl:output method="xml" version="1.1" encoding="utf-16"/>

<xsl:template name="main">
  <doc>
    <xsl:element name="r&#x133;k">
      <xsl:attribute name="R&#x132;k">RIJK</xsl:attribute>
      <bell>&#x7;</bell>
      <nbh>&#x83;</nbh>
      <nel>&#x85;</nel>
      <plain>vanilla</plain>
    </xsl:element>
  </doc>
</xsl:template>
 
</xsl:stylesheet>

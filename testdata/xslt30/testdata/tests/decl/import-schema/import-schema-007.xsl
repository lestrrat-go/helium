<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
default-validation="preserve">

  <!-- PURPOSE: Test validation on xsl:result-document. -->
  <?spec xslt#creating-result-trees?>
  <?spec xslt#validating-document-nodes?>

<xsl:template match="/">
  <xsl:result-document validation="preserve">
  <e>
  <e1 xsl:type="xs:ID">A001</e1>
  <e2 xsl:type="xs:string">A002</e2>
  <e3 xsl:type="xs:integer">003</e3>
  <e4 xsl:type="xs:date">2002-10-20</e4>
  </e>
  </xsl:result-document>
</xsl:template>            



</xsl:stylesheet>
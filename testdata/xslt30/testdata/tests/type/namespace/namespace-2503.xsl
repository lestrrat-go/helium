<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="2.0"
    xmlns:a="http://a.uri/"
    xmlns:b="http://b.uri/"
    exclude-result-prefixes="#all">

  <!-- test exclusion of a namespace that is actually required for one of the
       attributes of the literal result element. In Saxon 7.6.5, this causes
       the exclude-result-prefixes request to be ignored, so the output contains
       declarations of both xmlns:a="a.uri" and xmlns:a2="a.uri". In 7.7, this
       is changed so that the unwanted declaration for xmlns:a2 is now excluded -->
       
       <?spec xslt#lre-namespaces?>
  
  <!-- should copy c.uri only -->
  
  <xsl:template match="doc">
    <out a:foo="bar" xmlns:c="http://c.uri/" xmlns:d="http://d.uri/" xmlns:a2="http://a.uri/" xsl:exclude-result-prefixes="d a2"/>
  </xsl:template>
 
</xsl:stylesheet>

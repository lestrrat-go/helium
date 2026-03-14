<?xml version="1.0" encoding="iso-8859-1"?>
<?spec xslt#character-maps?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:output encoding="iso-8859-1"/>

  <!-- test use of a character map in attribute values -->

  <xsl:character-map name="map01">
    <xsl:output-character character="«" string="[&lt;]"/>
    <xsl:output-character character="»" string="[&gt;]"/>
    <xsl:output-character character="^" string="&quot;quote&quot;"/>    
  </xsl:character-map>
  
  <xsl:output method="xml" use-character-maps="map01"/>

  <xsl:template match="/">
    <a att1="normal" att2="&amp;«&amp;»&amp;" att3="&#100000;" att4="He said ^, I think"/>
  </xsl:template>

</xsl:stylesheet>
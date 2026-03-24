<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "cast as" on nodes. -->
  <?spec xpath#id-cast?>
  <?spec xpath#id-predefined-types?>
  
<xsl:variable name="data">
  <a>2002-02-02T12:00:00</a>
  <a>2002-02-03T12:00:00</a>
</xsl:variable>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="for $x in $data/a return $x cast as xs:dateTime " separator=","/></t>;
    </out> 
  </xsl:template>

</xsl:stylesheet>
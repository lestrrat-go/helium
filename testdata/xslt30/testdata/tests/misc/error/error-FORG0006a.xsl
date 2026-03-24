<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- PURPOSE: XPath 2.0: test min/max on a heterogeneous sequence -->
  <?spec fo#func-min?><?error FORG0006?>

  <xsl:template match="/">
    <xsl:variable name="list" 
    select="( xs:dateTime('1996-12-23T13:13:00.200+03:00'), 23, 'brian')"/>    
    <A>
        <xsl:value-of select="min($list)"/>;
        <xsl:value-of select="max($list)"/>;        
    </A>                               
  </xsl:template>

</xsl:stylesheet>
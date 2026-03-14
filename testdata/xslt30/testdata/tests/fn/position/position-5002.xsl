<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test last() in double predicate on a reverse axis.
        Second predicate does nothing, but can fool optimizations. -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="/">  
  <out>
    <xsl:for-each select="//center/ancestor::*[position() != last()][true()]">
      <xsl:value-of select="name()"/>
      <xsl:text>, </xsl:text>
    </xsl:for-each> 
  </out> 
</xsl:template>

</xsl:stylesheet>
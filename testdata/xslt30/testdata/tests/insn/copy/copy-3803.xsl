<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test siblings and parent axes work after deep copy -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>
  
<xsl:template match="/">
  <xsl:variable name="temp">
    <z1>
      <z2>
        <z3/>
        <xsl:copy-of select="."/>
        <z4/>
        <xsl:copy-of select="./*"/>
      </z2>
    </z1>
  </xsl:variable>
  <out>
    <z><xsl:value-of select="$temp//z3/following-sibling::node()/('['||local-name(.)||']')"/></z>
    <a><xsl:value-of select="deep-equal($temp//z3/following-sibling::node()/local-name(.), ('docs', 'z4', 'docs')) "/></a>
    <b><xsl:value-of select="every $n in $temp//z2/node() satisfies $n/parent::* is $temp//z2"/></b>
    <c><xsl:value-of select="deep-equal($temp//z2/node()[last()]!reverse(preceding-sibling::node()/local-name(.)), ('z4', 'docs', 'z3'))"/></c>
  </out>
</xsl:template>

</xsl:stylesheet>
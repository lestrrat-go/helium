<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:variable name="lastMove">
  <xsl:for-each select="/Nodes/Black">
    <xsl:if test="position()=1">
      <xsl:value-of select="@number"/>
    </xsl:if>
  </xsl:for-each>
</xsl:variable>

<xsl:template match="/">
  <out>
    <title>
      <xsl:text>Reproducing Bugzilla#6284: predicate/global variable/position()</xsl:text>
    </title>
    <xsl:text>
</xsl:text>
    <xsl:for-each select="/Nodes/Black[number(@number) &lt;= number($lastMove)]">
      <duplicate><!-- Should get one of these -->
        <xsl:text>found a duplicate at </xsl:text>
        <xsl:value-of select="position()"/>
      </duplicate>
    </xsl:for-each>
    <xsl:text>
</xsl:text>
    <last><!-- Should be 1 -->
      <xsl:value-of select="$lastMove"/>
    </last>
  </out>
</xsl:template>

</xsl:stylesheet>
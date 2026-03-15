<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: test union operator using overlapping node-sets. Results should
       always be output in doc order regardless of order of select attribute. -->
  
<xsl:key name="which" match="child" use="@wide|@deep"/>
<xsl:key name="one" match="child" use="@deep"/>
<xsl:key name="two" match="child" use="@wide"/>

<xsl:template match="doc">
  <out><xsl:text>
</xsl:text>
<xsl:apply-templates select = "child[@wide='3']|child[@deep='3']"/>,
<xsl:apply-templates select = "child[@deep='3']|child[@wide='3']"/>,
<xsl:apply-templates select = "key('which','3')"/>,
<xsl:apply-templates select = "key('one','3') | key('two','3')"/>,
<xsl:apply-templates select = "child[@wide='3'] | key('one','3')"/>,
<xsl:apply-templates select = "key('two','3') | document('select-50.xml')/child[@wide='3'] | child[@deep='3']"/>,
</out>
</xsl:template>

<xsl:template match = "*">
<xsl:value-of select = "."/>
</xsl:template>

</xsl:stylesheet>
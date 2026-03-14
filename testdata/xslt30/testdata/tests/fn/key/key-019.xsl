<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Purpose: Test for variable as first argument to key(). -->


<xsl:variable name="keysp">bigspace</xsl:variable>

<xsl:key name="bigspace" match="div" use="subdiv/title" />
<xsl:key name="smallspace" match="subdiv" use="title" />
<xsl:key name="filterspace" match="div[@allow='yes']" use="subdiv/title" />

<xsl:template match="doc">
 <root>
  Using keyspace <xsl:value-of select="$keysp"/>...
  <xsl:value-of select="key($keysp, 'Introduction')/subdiv/p"/>
  <xsl:value-of select="key($keysp, 'Stylesheet Structure')/subdiv/p"/>
  <xsl:value-of select="key($keysp, 'Expressions')/subdiv/p"/>
 </root>
</xsl:template>

</xsl:stylesheet>

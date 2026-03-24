<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- Purpose: In XSLT 3.0, namespace nodes can match patterns so they can be used in keys. -->

  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <xsl:key name="ns" match="element() | attribute() | namespace-node()" use="name()"/>

  <xsl:template match="/">
    <out>
      <xsl:for-each select="key('ns', 'ext')">
        <e><xsl:value-of select="name(..)"/></e>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>

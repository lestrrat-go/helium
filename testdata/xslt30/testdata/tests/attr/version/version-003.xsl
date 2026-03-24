<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="8.5">

  <!-- PURPOSE: Test xsl:message. -->
  <?spec xslt#message?>
  <xsl:template match="/">
    <out>
        <xsl:message>A simple text message&#xa;</xsl:message>
        <xsl:message>A non-ASCII message (Straßenüberführung) &#xa;</xsl:message>
        <xsl:message terminate="no">An <b>XML</b> message&#xa;</xsl:message>
        <xsl:message><html><body>An HTML message <td nowrap="nowrap"/>.&#xa;</body></html></xsl:message>
        <xsl:message terminate="yes">Finishing now, bye!&#xa;</xsl:message>
    </out>
  </xsl:template>
   
</xsl:stylesheet>

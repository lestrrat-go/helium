<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: basic numeric formats -->

  <xsl:param name="d" as="xs:date" select="xs:date('2003-09-07')"/>

  <xsl:template name="main">
<out>;
 <tr><code>2003-09-7</code><code x='{format-date($d,"[Y]-[M01]-[D]")}'/></tr>;
 <tr><code>9-7-2003</code><code x='{format-date($d,"[M]-[D]-[Y]")}'/></tr>;
 <tr><code>7-9-2003</code><code x='{format-date($d,"[D]-[M]-[Y]")}'/></tr>;
 <tr><code>7 IX 2003</code><code x='{format-date($d,"[D1] [MI] [Y]")}'/></tr>;
 <tr><code>[2003-09-07]</code><code x='{format-date($d,"[[[Y]-[M01]-[D01]]]")}'/></tr>;
 <tr><code>[2003-09-07]</code><code x='{format-date($d,"[[[Y0001]-[M01]-[D01]]]")}'/></tr>; 
 <tr><code>(03-09-07)</code><code x='{format-date($d,"([Y01]-[M01]-[D01])")}'/></tr>; 
</out>     
  </xsl:template>

</xsl:stylesheet>
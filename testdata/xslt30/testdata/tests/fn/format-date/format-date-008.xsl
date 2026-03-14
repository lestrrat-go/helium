<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: numeric formats using Osmanya (non-BMP) digits -->
  <!-- Note: Saxon 8.7 can't handle this one -->
  
  <xsl:output encoding="iso-8859-1"/>

  <xsl:param name="d" as="xs:date" select="xs:date('2003-09-07')"/>

  <xsl:template name="main">
<out>;
 <tr><code>2003-09-7</code><code x='{format-date($d,"[Y&#x104a0;&#x104a0;&#x104a0;&#x104a1;]-[M&#x104a0;&#x104a1;]-[D&#x104a1;]")}'/></tr>;
 <tr><code>9-7-2003</code><code x='{format-date($d,"[M&#x104a1;]-[D&#x104a1;]-[Y&#x104a0;&#x104a0;&#x104a0;&#x104a1;]")}'/></tr>;
 <tr><code>(03-09-07)</code><code x='{format-date($d,"([Y&#x104a0;&#x104a1;]-[M&#x104a0;&#x104a1;]-[D&#x104a0;&#x104a1;])")}'/></tr>; 
</out>     
  </xsl:template>

</xsl:stylesheet>
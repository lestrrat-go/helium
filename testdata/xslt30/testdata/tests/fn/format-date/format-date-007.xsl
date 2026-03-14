<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: numeric formats using Thai digits -->
  
  <xsl:output encoding="iso-8859-1"/>

  <xsl:param name="d" as="xs:date" select="xs:date('2003-09-07')"/>

  <xsl:template name="main">
<out>;
 <tr><code>2003-09-7</code><code x='{format-date($d,"[Y&#xe50;&#xe50;&#xe50;&#xe51;]-[M&#xe50;&#xe51;]-[D&#xe51;]")}'/></tr>;
 <tr><code>9-7-2003</code><code x='{format-date($d,"[M&#xe51;]-[D&#xe51;]-[Y&#xe50;&#xe50;&#xe50;&#xe51;]")}'/></tr>;
 <tr><code>(03-09-07)</code><code x='{format-date($d,"([Y&#xe50;&#xe51;]-[M&#xe50;&#xe51;]-[D&#xe50;&#xe51;])")}'/></tr>; 
</out>     
  </xsl:template>

</xsl:stylesheet>
<?xml version="1.0"?> 
<xsl:package name="urn:package:forwards-011" package-version="4.0.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
  <!-- 
  The spec says:
  
  If the element is in the XSLT namespace and appears as a child of the xsl:stylesheet element, 
  and XSLT 3.0 does not allow the element to appear as a child of the xsl:stylesheet element, 
  then the element and its content must be ignored.
  
  This isn't directly relevant here: xsl:expose isn't allowed as a child of xsl:stylesheet,
  and it doesn't appear here as a child of xsl:stylesheet. But this makes it a bit of a special
  case, so it's worth testing that all is well.
  -->
  
  <xsl:template name="go">
    <out>ok</out>
  </xsl:template>
  
  <xsl:expose version="23.0"
    component="template" names="go" visibility="public"/>
  
  
</xsl:package>

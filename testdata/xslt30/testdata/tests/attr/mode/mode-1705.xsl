<xsl:package
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  declared-modes="yes"
  default-mode="a">
  
  <!-- this mode 'a' is an eligible initial mode, it is in @default-mode, whether it is private or (here) public does not matter --> 
  <xsl:mode name="a" visibility="public" />
  
  <!-- mode '#unnamed is always an eligible initial mode -->
  <xsl:mode/>  
  
  <!-- a private mode (by default), is not an eligible default mode -->
  <xsl:mode name="private-mode" />

  <xsl:template match="/">
      <ok/>
  </xsl:template>  
  
  <xsl:template match="/" mode="#unnamed">
    <ok-unnamed/>
  </xsl:template>

</xsl:package>   
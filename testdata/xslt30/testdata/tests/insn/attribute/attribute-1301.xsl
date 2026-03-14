<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
            xmlns:bdd="http://bdd.test.com">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: The attribute must be in the designated namespace, even if the prefix has
    to be reset or ignored. -->
    
  <!-- MHK: moreover, in XSLT 2.0, the "jam" element must have an in-scope namespace for 
      xmlns:bdd="http://bdd.test.com", and this takes precedence over the use of the prefix
      bdd in the attribute name. -->  

<xsl:template match="/">
  <xsl:variable name="temp">
  <out>
    <jam>
      <xsl:attribute name="{docs/b}" namespace="http://xyz.com">jaminben</xsl:attribute>
    </jam>
  </out>
  </xsl:variable>
  <ans bdd="{namespace-uri-for-prefix('bdd', $temp/out/jam)}" attr-ns="{namespace-uri($temp/out/jam/@*:attr)}"/>
</xsl:template>

</xsl:stylesheet>
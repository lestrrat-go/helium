<?xml version='1.0'?>
<!DOCTYPE xsl:stylesheet [
<!ENTITY child SYSTEM "dir/use-when-0136.ent">
]>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

&child;

  <!-- PURPOSE: test that xml:base affects the static base URI of a static expression. -->
    
  <xsl:template name="main" use-when="ends-with($sbu, 'dir/use-when-0136.ent')">
    <ok/>
  </xsl:template>
  
</xsl:stylesheet>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Crashes with an NPE in Saxon 7.9.1 -->
<?spec xslt#fallback?>

<xsl:template match="/">
<a>
     <xsl:value-of
       xmlns:ext="file://ext"
       extension-element-prefixes="ext">
       <ext:instruction>
         <xsl:fallback
           xmlns:nsexcl="file://nsexcl"
           xmlns:nsexcl2="file://nsexcl2"
           xmlns:ns="file://ns"
           xmlns:ns2="file://ns2"
           exclude-result-prefixes="nsexcl nsexcl2">
           <!-- the <dummy/> causes the NPE -->
           <dummy nsexcl2:dummy="abc" ns2:dummy="def">12</dummy>
         </xsl:fallback>
       </ext:instruction>
     </xsl:value-of>
</a>
</xsl:template>

</xsl:stylesheet>
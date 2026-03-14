<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
   xmlns:a="http://uri1.com/" xmlns:b="http://uri2.com/">


   <?spec xslt#lre-namespaces?>
   <!-- BUG: Saxon 5.3.1/006 -->
   <!-- DESCRIPTION: When a namespace is excluded from the result tree using xsl:exclude-result-prefixes,
    but is then used on a literal result element, the request to exclude it should be
    ignored, to ensure that the result is well-formed XML following the Namespace rules.
    Currently the request to exclude the namespace is ignored if the namespace is used on
    the element containing the xsl:exclude-result-prefixes attribute, but not if it is used
    on an inner element. -->

   <xsl:template match="/">
      <out changed="2004-04-06">
         <a:aa>
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa xsl:exclude-result-prefixes="a">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa xsl:exclude-result-prefixes="b">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa xsl:exclude-result-prefixes="a b">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa b:bbb="b">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa b:bbb="b" xsl:exclude-result-prefixes="a">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa b:bbb="b" xsl:exclude-result-prefixes="b">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
         <a:aa b:bbb="b" xsl:exclude-result-prefixes="a b">
            <b:AA/>
            <x b:bbb="b"/>
         </a:aa>
      </out>
   </xsl:template>

</xsl:stylesheet>

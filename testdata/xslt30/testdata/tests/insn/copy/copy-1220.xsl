<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="3.0"
  expand-text="yes"
  exclude-result-prefixes="#all">

<?spec xslt#copy-of?>    
    <!-- Purpose: Test copy-of, tree-to-tree, with various namespace options -->

  <xsl:param name="COPY_NS" as="xs:boolean" static="yes" required="yes"/>
  
  <xsl:variable name="fragment">
    <wrapper xmlns:w="http://www.w.com/">
      <a xmlns:a="http://www.a.com/" a:att="A">
        <aa xmlns="http://www.aa.com/"/>
      </a>
      <xsl:text>sandwich</xsl:text>
      <a xmlns="http://www.a.com/">
        <aa xmlns:aa="http://www.aa.com/"/>
      </a>
    </wrapper>
  </xsl:variable>

<xsl:template name="xsl:initial-template">
  <xsl:variable name="result">
    <doc xmlns="http://www.out.com/">
      <xsl:copy-of select="$fragment/wrapper/child::node()" _copy-namespaces="{$COPY_NS}"/>
    </doc>
  </xsl:variable>
  <out>
    <m>{Q{f}namespaces($result/*/*[1])}</m>
    <n>{Q{f}namespaces($result/*/*[1]/*[1])}</n>
    <o>{Q{f}namespaces($result/*/*[2])}</o>
    <p>{Q{f}namespaces($result/*/*[2]/*[1])}</p>
  </out>
</xsl:template>
  
<xsl:function name="f:namespaces" as="xs:string" xmlns:f="f">
  <xsl:param name="node" as="element()"/>
  <xsl:sequence select="name($node) || ' : ' || string-join(sort($node/namespace::*[name() ne 'xml']/(name()||'='||.||'; ')))"/>
</xsl:function>  

</xsl:stylesheet>

<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
      xmlns:java="http://xml.apache.org/xslt/java"
	xmlns:bdd="http://buster.com"
	xmlns:jad="http://administrator.com"
	xmlns="http://www.lotus.com/">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: Test exclude-result-prefixes as an attribute on an LRE. -->
  <!-- Elaboration: The created element node will also have a copy of the namespace nodes 
       that were present on the element node in the stylesheet tree with the exception 
       of any namespace node whose string-value is the XSLT namespace URI, a namespace 
       URI declared as an extension namespace, or a namespace URI designated as an 
       excluded namespace. (Can't exclude namespaces that are used, however.)
    The default namespace is used, and so can't be excluded. -->

<xsl:template match="doc">
  <out english="to leave" xsl:exclude-result-prefixes="java jad #default"/>
</xsl:template>

</xsl:stylesheet>

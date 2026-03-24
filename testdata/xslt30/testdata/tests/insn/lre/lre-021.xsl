<?xml version="1.0"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:ext="http://somebody.elses.extension"
    xmlns:java="http://xml.apache.org/xslt/java"
    xmlns:jad="http://administrator.com"
    xmlns="http://www.lotus.com/"
    xmlns:ped="http://tester.com"
    xmlns:bdd="http://buster.com"
    extension-element-prefixes="ext"
    exclude-result-prefixes="java jad #default">

<?spec xslt#attributes-for-lres?>
<?error XTSE0085?>
    <!-- AdditionalSpec: 2.1 of XSLT for namespaces on attributes -->
  <!-- Purpose: Testing the xsl:transform element and its attributes. english
       attribute and #default,ped,bdd namespace nodes are all that should be output.
       (#default is used.) xsl:if must be assumed to be a directive to the processor,
       so it could raise an error. -->

<xsl:template match="doc">
  <out xsl:if= "my if" english="to leave"/>
</xsl:template>

</xsl:transform>
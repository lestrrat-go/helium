<?xml version="1.0" encoding="UTF-8"?>
<xsl:package 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    name="http://www.160.com/common"
    package-version="0.1"
    version="3.0">
        
    <xsl:use-package name="http://www.160.com/logger" package-version="0.1">
        <xsl:accept component="*" names="*" visibility="public"/> 
    </xsl:use-package>
    <xsl:use-package name="http://www.160.com/generalFunctions" package-version="0.1">
        <xsl:accept component="*" names="*" visibility="public"/>
    </xsl:use-package>
    <xsl:use-package name="http://www.160.com/generalVariables" package-version="0.1">
        <xsl:accept component="*" names="*" visibility="public"/>
    </xsl:use-package>
    <xsl:use-package name="http://www.160.com/runner" package-version="0.1">
        <xsl:accept component="*" names="*" visibility="public"/>
    </xsl:use-package>
    
    
</xsl:package>